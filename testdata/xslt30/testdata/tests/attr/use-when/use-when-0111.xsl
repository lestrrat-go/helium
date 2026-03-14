<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test use-when with an extension function. -->
  <?spec xslt#conditional-inclusion?>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para" xmlns:not-saxon="http://not-saxon.sf.net/" exclude-result-prefixes="not-saxon">
    <p><xsl:value-of select="not-saxon:line-number()" use-when="function-available('not-saxon:line-number')"/></p>
  </xsl:template>

</xsl:stylesheet>