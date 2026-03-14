<?xml version="1.0"?>
<xsl:stylesheet
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error ?>
    <!-- Purpose: Test error when xmlns prefix or corresponding URI is abused -->

<xsl:template match = "data">
  <xsl:param name="prefix" select="'a'"/>
  <xsl:param name="one" select="1"/>
  <out>
    <xsl:namespace name="p" select="concat('http://www.w3.org/', 1999+$one, '/xmlns/')"/>
  </out>
</xsl:template>

</xsl:stylesheet>