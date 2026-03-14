<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns="http://testguys.com/">

<?spec xslt#xsl-element?>
    <!-- Purpose: Use prefixed name; requested NS matches default rather than what's declared for that prefix. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="p2:foo" namespace="http://testguys.com/" xmlns:p2="http://barz.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>