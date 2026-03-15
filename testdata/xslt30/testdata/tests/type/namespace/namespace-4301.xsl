<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:n="http://ns.test.com">

<?spec xslt#copy-of?>
    <!-- Purpose: Test for resetting of a namespace prefix by copy-of. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match = "/">
  <out>
    <n:x>from stylesheet</n:x>
    <xsl:element name="e" namespace="http://literalURI">
      <xsl:copy-of select="doc/*"/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>