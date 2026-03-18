<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:cat="http://www.w3.org/2012/10/xslt-test-catalog"
  exclude-result-prefixes="cat">
  
  <!-- Test case names must be unique -->

<xsl:template match="/">
      <out>
        <xsl:variable name="tests" select="/*/cat:test-set/document(@file)/*/cat:test-case"/>
        <total all="{count($tests)}" need30="{count($tests[contains(cat:dependencies/cat:spec/@value, 'XSLT30') or contains(../cat:dependencies/cat:spec/@value, 'XSLT30')])}"/> 
      </out>
</xsl:template>

</xsl:stylesheet>