<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
  
  <!-- streaming, xsl:for-each-group group-adjacent, composite grouping key, 
       get min and max using xsl:iterate with position() and xsl:break -->
  
  <xsl:mode name="s" streamable="yes"/>
    
  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
      <xsl:apply-templates select="account" mode="s"/>
    </xsl:source-document>
  </xsl:template> 


  <xsl:template match="account" mode="s">
     <out>
      <xsl:for-each-group select="transaction"
         group-adjacent="year-from-date(xs:date(@date)), format-date(xs:date(@date), '[W]')"
         composite="yes">
         <batch year="{current-grouping-key()[1]}" week="{current-grouping-key()[2]}">
            <xsl:iterate select="current-group()">
              <xsl:param name="min" select="+1000000000"/>
              <xsl:param name="max" select="-1000000000"/>
              <xsl:on-completion>
                <min><xsl:value-of select="$min"/></min>
                <max><xsl:value-of select="$max"/></max>
              </xsl:on-completion>
              <xsl:choose>
                <xsl:when test="position() = 3">
                  <xsl:break>
                    <min><xsl:value-of select="$min"/></min>
                    <max><xsl:value-of select="$max"/></max>
                  </xsl:break>
                </xsl:when>
                <xsl:otherwise>
                  <xsl:next-iteration>
                    <xsl:with-param name="min" select="min(($min, xs:decimal(@value)))"/>
                    <xsl:with-param name="max" select="max(($max, xs:decimal(@value)))"/>
                  </xsl:next-iteration>
                </xsl:otherwise>
              </xsl:choose>
            </xsl:iterate>
         </batch>
      </xsl:for-each-group> 
    </out>
  </xsl:template>   
  

       
</xsl:transform>

