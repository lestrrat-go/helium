<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
  
  <!-- within a streaming template, use xsl:for-each-group group-adjacent with count() -->
  
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
         <batch date="{current-grouping-key()}">
            <size><xsl:value-of select="count(current-group())"/></size>
         </batch>
      </xsl:for-each-group> 
    </out>
  </xsl:template>   
  

       
</xsl:transform>

