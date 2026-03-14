<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<!-- Use xsl:copy to copy document nodes -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">
    <xsl:template match="/">
      <xsl:variable name="doc">23</xsl:variable>
      <xsl:variable name="docs" as="document-node()*">
        <xsl:for-each select="$doc">
          <xsl:copy><a>24</a></xsl:copy>
          <xsl:copy><a>25</a></xsl:copy>
          <xsl:copy><a>26</a></xsl:copy>
        </xsl:for-each>
      </xsl:variable>
      <out count="{count($docs)}">
        <xsl:sequence select="$docs"/>
      </out>
    </xsl:template>
</xsl:stylesheet>
