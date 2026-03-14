<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#keys?>
    <!-- Purpose: Test xsl:key where match nodes occur at different levels in the document. -->

<xsl:key name="mykey" match="div" use="title" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="count(key('mykey', 'Introduction'))"/>
  </out>
</xsl:template>

</xsl:stylesheet>
