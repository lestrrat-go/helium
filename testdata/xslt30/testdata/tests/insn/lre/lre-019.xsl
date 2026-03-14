<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://lre-019.com/" exclude-result-prefixes="f">

<?spec xslt#lre-namespaces?>

    <!-- Purpose: Test use of xsl:on-empty on a LRE. -->
  <!-- Elaboration: Some simple cases where on-empty fires or does not fire -->
  
<xsl:param name="zls" select="''"/>  

<xsl:template match="/">
  <out>
    <a><xsl:where-populated><lre>12</lre></xsl:where-populated></a>
    <b><xsl:where-populated><lre><x/></lre></xsl:where-populated></b>
    <c><xsl:where-populated><lre><!--heh--></lre></xsl:where-populated></c>
    <d><xsl:where-populated><lre><?pi?></lre></xsl:where-populated></d>
    <e><xsl:where-populated><lre><xsl:attribute name="x">12</xsl:attribute></lre></xsl:where-populated></e>
    <f><xsl:where-populated><lre x="12"/></xsl:where-populated></f>
    <g><xsl:where-populated><lre><xsl:attribute name="x" select="'abc'"/></lre></xsl:where-populated></g>
    
    <p><xsl:where-populated><lre/></xsl:where-populated></p>
    <q><xsl:where-populated><lre></lre></xsl:where-populated></q>
    <r><xsl:where-populated><lre><xsl:value-of select="$zls"/></lre></xsl:where-populated></r>
    <s><xsl:where-populated><lre xmlns:p="http://p.com/"></lre></xsl:where-populated></s>
  </out>
</xsl:template>

<xsl:function name="f:att">
  <xsl:attribute name="x" select="'abc'"/>
</xsl:function>

</xsl:stylesheet>