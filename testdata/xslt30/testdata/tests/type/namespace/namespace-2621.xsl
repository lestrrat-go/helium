<?xml version="1.0"?>
<xsl:stylesheet
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error ?>
    <!-- Purpose: Test error when xmlns prefix is abused -->

<xsl:template match = "data">
  <xsl:param name="prefix" select="'a'"/>
  <out>
    <xsl:element name="{$prefix}:foo" namespace="{'http://www.w3.org'}/{1999+1}/{'xmlns'}/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>