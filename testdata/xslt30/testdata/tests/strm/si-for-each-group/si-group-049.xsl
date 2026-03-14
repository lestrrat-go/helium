<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
   
  <!-- within a streaming template, use xsl:for-each-group group-adjacent, with a variable evaluated once per group -->
  
  <xsl:mode name="s" streamable="yes"/>
       
    
  <xsl:template name="xsl:initial-template">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
    	<xsl:apply-templates select="account" mode="s"/>
    </xsl:source-document>
  </xsl:template> 


  <xsl:template match="account" mode="s">
     <out>
      <xsl:for-each-group select="transaction" group-adjacent="@date">
         <xsl:variable name="tomorrow" select="xs:date(current-grouping-key()) + xs:dayTimeDuration('P1D')"/>
         <xsl:variable name="values" select="current-group()/(+@value)"/>
         <date tomorrow="{$tomorrow}" total="{sum($values)}">
           <values when="{$tomorrow}"><xsl:value-of select="$values"/></values>
         </date>
      </xsl:for-each-group> 
    </out>
  </xsl:template>   
       
</xsl:transform>

