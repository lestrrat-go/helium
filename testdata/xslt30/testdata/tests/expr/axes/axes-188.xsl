<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-predicates?>
  <!-- Purpose: Test to select the set of all aunts, great-aunts, etc. but exclude ancestors. -->
  <!-- "ancestor::*[count(child::*) &gt; 1]" is the set of all ancestors with other children. -->
  
  <!-- MHK: the test is incorrect in using "=" to test for node identity, but never mind... -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:apply-templates 
      select="ancestor::*[count(child::*) &gt; 1]/*[not(. = current()/ancestor-or-self::*)]"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:text> </xsl:text><xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>
