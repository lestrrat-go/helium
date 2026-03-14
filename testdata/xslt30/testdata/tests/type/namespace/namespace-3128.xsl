<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test for namespace attribute not (previously) tied to prefix. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="q:foo" namespace="http://testguys.com">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>