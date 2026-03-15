<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:bdd="http://buster.com"
    xmlns:jad="http://administrator.com">

<?spec xslt#lre-namespaces?>
    <!-- Purpose: The designation of a namespace as an excluded namespace is 
       effective within the subtree of the stylesheet rooted at the element 
       bearing the exclude-result-prefixes or xsl:exclude-result-prefixes attribute. -->

<xsl:template match="doc">
  <out x="by the corner"><xsl:text>&#010;</xsl:text>
  <sits x="little jack horner" xsl:exclude-result-prefixes="jad"/><xsl:text>&#010;</xsl:text> 
  <minding x="his peas and queues" xsl:exclude-result-prefixes="jad bdd"/></out>
</xsl:template>

</xsl:stylesheet>
