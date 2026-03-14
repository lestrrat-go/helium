<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:a="http://foo.com/">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test that exclude-result-prefixes should NOT affect xsl:element (when prefix needed) -->

<xsl:template match="/">
  <out xmlns:b="http://barz.com/" xsl:exclude-result-prefixes="a b">
    <xsl:element name="b:mid">
      <xsl:element name="a:inner"/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>