<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
   
  <!-- Filter current-group() by position(), with where-populated -->
  <!-- Same as si-group-042, but using xsl:sequence rather than xsl:copy-of -->
  
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
        <xsl:where-populated>
          <group key="{current-grouping-key()}">
            <xsl:sequence select="current-group()[2]"/>         
          </group>
        </xsl:where-populated>
      </xsl:for-each-group> 
    </out>
  </xsl:template>   
       
</xsl:transform>

