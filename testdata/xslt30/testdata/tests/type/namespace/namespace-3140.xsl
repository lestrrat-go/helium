<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns="http://testguys.com/">

<?spec xslt#xsl-element?>
    <!-- Purpose: Use prefixed xmlns declaration with prefixed name; default was set. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="p2:foo" xmlns:p2="http://barz.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>