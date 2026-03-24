<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0"
  exclude-result-prefixes="xs">

  <!-- Test use of local variable declared locally within a predicate of xsl:number/@count -->

  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/foo">
        <xsl:number count="foo[let $n := number(@bar) return . = $n*$n]" level="any"/>
        <xsl:text>|</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
