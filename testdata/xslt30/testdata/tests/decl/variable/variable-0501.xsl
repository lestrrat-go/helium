<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="2.0">
 
  <!-- Test case for Saxon bug 2980277 - xsl:value-of generating multiple text nodes -->      
       
  <xsl:strip-space elements="*"/>
    
  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="//book"/>
    </out>
  </xsl:template>
   
  <xsl:template match="book">
    <book>
       <xsl:variable name="temp" as="text()*">
         <xsl:value-of select="( .//chtitle | .//@*)"/>
       </xsl:variable>
       <nodes><xsl:value-of select="count($temp)"/></nodes>
       <xsl:if test="position() = 1">
         <copy><xsl:for-each select="$temp"><t><xsl:value-of select="."/></t></xsl:for-each></copy>
        </xsl:if>
    </book>
  </xsl:template>
    
</xsl:transform>

