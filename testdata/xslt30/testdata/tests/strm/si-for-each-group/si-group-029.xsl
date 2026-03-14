<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs">

<xsl:mode streamable="yes"/>
<xsl:strip-space elements="*"/>

<xsl:template match="root">
  <xsl:copy>
    <xsl:for-each-group select="record/copy-of()" group-adjacent="foo">
      <group key="{current-grouping-key()}">
        <xsl:copy-of select="current-group()"/>
      </group>
    </xsl:for-each-group>
  </xsl:copy>
</xsl:template>

</xsl:stylesheet>