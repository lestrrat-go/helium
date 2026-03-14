<xsl:stylesheet exclude-result-prefixes="#all" version="3.0" 
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Saxon bug 3791 - reference to $err:code in a predicate -->

  <xsl:template name="main">
    <out>
      <xsl:variable as="xs:QName" name="my-err" select="xs:QName('my-err-local')" />
      <xsl:try select="error($my-err)">
        <xsl:catch>
          <xsl:value-of select="'test:', 'Bang!'[$err:code eq $my-err]" />
        </xsl:catch>
      </xsl:try>
    </out>  
  </xsl:template>
</xsl:stylesheet>