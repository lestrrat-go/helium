<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Mix a non-empty xmlns declaration and namespace attrib (different URI) in xsl:element. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="foo" namespace="http://abc.com/" xmlns="http://test.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


