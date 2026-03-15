<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Purpose: Test retrieving correct values from node-set variables by positional predicate. -->
 
  <xsl:output method="xml"/>

  <xsl:template match="/">
    <xsl:variable name="x1">
      <xsl:variable name="node" select="/foo/*/test/text()"/>
      <xsl:value-of select="$node[1]"/>
    </xsl:variable>
    <xsl:variable name="x6">
      <xsl:variable name="node" select="/foo/*/test/text()"/>
      <xsl:for-each select="$node[6]">
        <xsl:value-of select="."/>
      </xsl:for-each>
    </xsl:variable>
    <xsl:variable name="xL">
      <xsl:variable name="node" select="/foo/*/test/text()"/>
      <xsl:for-each select="$node[last()]">
        <xsl:value-of select="."/>
      </xsl:for-each>
    </xsl:variable>
    <out>
      <x1><xsl:value-of select="$x1"/></x1><xsl:text>&#10;</xsl:text>
      <x6><xsl:value-of select="$x6"/></x6><xsl:text>&#10;</xsl:text>
      <xL><xsl:value-of select="$xL"/></xL>
    </out>
  </xsl:template>

</xsl:stylesheet>