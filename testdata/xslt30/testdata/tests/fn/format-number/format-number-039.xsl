<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"> 

 <!-- Purpose: Test of multiple decimal-format elements with identical names.
    This is allowed as long as all attributes are identical (including defaults). -->

      
  <!-- Runs with version="1.0" -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number('NaN','###','decimal2')"/>
    <xsl:text>, </xsl:text>
    <xsl:value-of select="format-number(3.2,'###.0','decimal2')"/>
  </out>
</xsl:template>

<xsl:decimal-format name="decimal2" zero-digit='0' NaN='not a number'/>
<xsl:decimal-format name="decimal2" NaN='not a number' decimal-separator='.'/>

</xsl:stylesheet>
