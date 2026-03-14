<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
   
  <!-- Access position() within current-group() -->
  
  <xsl:mode name="s" streamable="yes"/>
       
    
  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
    	<xsl:apply-templates select="account" mode="s"/>
    </xsl:source-document>
  </xsl:template> 


  <xsl:template match="account" mode="s">
     <out>
      <xsl:for-each-group select="transaction"
         group-adjacent="@date">
         <group key="{current-grouping-key()}">
           <xsl:for-each select="current-group()">
             <xsl:value-of select="position()"/>
           </xsl:for-each>          
         </group>
      </xsl:for-each-group> 
    </out>
  </xsl:template>   
       
</xsl:transform>

