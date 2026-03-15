<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#xsl-element?>
    <!-- Purpose: Put both an unprefixed xmlns declaration and namespace attrib in xsl:element. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="foo" namespace="http://www.zebie.com/" xmlns="">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


