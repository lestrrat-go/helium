<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns="http://zilch.com/" xmlns:p1="http://testguys.com/">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test for prefixed name when prefixed NS is in scope; also set default for stylesheet. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="p1:foo">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>