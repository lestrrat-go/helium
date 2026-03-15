<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:n="http://ns.test.com"
  xmlns:s="http://example.com">

<?spec xslt#copy-of?>
    <!-- Purpose: Test copying of a namespace node by copy-of. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match = "/">
  <out>
    <n:x>from stylesheet</n:x>
    <xsl:element name="e" namespace="http://literalURI">
      <xsl:copy-of select="doc/s:a"/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>