<xsl:stylesheet version="3.0"
  xmlns:fn="http://www.w3.org/2005/xpath-functions"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs fn">

  <!--  call position() when context item is undefined within xsl:evaluate -->


  <xsl:param name="e3">fn:position()</xsl:param>

  <xsl:template match="/">
    <out>
      <a>
        <xsl:evaluate xpath="($e3)"/>
      </a>
    </out>

  </xsl:template>

</xsl:stylesheet>
