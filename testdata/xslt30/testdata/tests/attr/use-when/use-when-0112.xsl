<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test use-when with an extension function. -->
  <?spec xslt#conditional-inclusion?>
  <?dependency saxon:line-number?>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para" xmlns:saxon="http://not.saxon.sf.net/">
    <p><xsl:value-of select="saxon:line-number()" use-when="function-available('saxon:line-number')"/></p>
  </xsl:template>

</xsl:stylesheet>