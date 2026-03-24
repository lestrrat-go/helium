<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0" 
    xmlns:foo="http://foo.com"
    xmlns:baz="http://foo.com"
    exclude-result-prefixes="foo baz">

  <!-- Purpose: Test of multiple decimal-format elements with identical qualified names.
    This is allowed as long as all attributes are identical (including defaults). -->
  <!-- Runs with version="1.0" -->

<xsl:decimal-format name="foo:decimal1" minus-sign='-' NaN='not a number'/>
<xsl:decimal-format name="baz:decimal1" NaN='not a number' decimal-separator='.'/>

<xsl:template match="doc">
  <out>
    <xsl:text>&#10;</xsl:text>
    <foo>
      <xsl:value-of select="format-number('NaN','###','foo:decimal1')"/>
      <xsl:text>, </xsl:text>
      <xsl:value-of select="format-number(-13.2,'###.0','foo:decimal1')"/>
    </foo>
    <xsl:text>&#10;</xsl:text>
    <baz>
      <xsl:value-of select="format-number('NaN','###','baz:decimal1')"/>
      <xsl:text>, </xsl:text>
      <xsl:value-of select="format-number(-13.2,'###.0','baz:decimal1')"/>
    </baz>
  </out>
</xsl:template>

</xsl:stylesheet>
