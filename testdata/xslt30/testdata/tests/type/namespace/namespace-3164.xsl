<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Use prefixed xmlns declaration, namespace attrib sets different URI of that prefix. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="p2:foo" namespace="http://barz.com/" xmlns:p2="http://testguys.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>