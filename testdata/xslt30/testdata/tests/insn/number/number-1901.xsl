<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0"
  exclude-result-prefixes="xs">

  <!-- Test use of current() function in a predicate of xsl:number/@count -->

  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/foo">
        <xsl:variable name="n" as="xs:integer">
          <xsl:number count="foo[@bar = current()/@bar]"/>
        </xsl:variable>
        <xsl:value-of select="concat($n, '|')"/>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
