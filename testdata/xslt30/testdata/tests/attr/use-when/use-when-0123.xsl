<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: ERROR: test use-when attempting to access context node. -->
  <?spec xslt#conditional-inclusion?>
  <?error xxxxxxxx?>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para">
    <p><xsl:value-of select="." use-when=". != 17"/></p>
  </xsl:template>

</xsl:stylesheet>