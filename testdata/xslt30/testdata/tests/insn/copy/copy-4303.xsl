<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<!-- Use xsl:copy-of to copy document nodes -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">
    <xsl:template match="/">
      <xsl:variable name="doc"><a>23</a></xsl:variable>
      <xsl:variable name="docs" as="document-node()*">
        <xsl:copy-of select="$doc, $doc, $doc"/>
      </xsl:variable>
      <out count="{count($docs)}">
        <xsl:sequence select="$docs"/>
      </out>
    </xsl:template>
</xsl:stylesheet>
