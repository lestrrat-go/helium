<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
      xmlns:tst="http://test/"
      exclude-result-prefixes="tst">

<?spec fo#func-namespace-uri?>
    <!-- Purpose: Test of namespace-uri() on default namespace declaration. -->
  <!-- Part 4 of the Namespaces in XML spec says "The prefix xmlns is used only for
     namespace bindings and is not itself bound to any namespace name. -->

<xsl:template match="/tst:a">
  <out>
   <xsl:value-of select="namespace-uri(namespace::*[string()='http://www.w3.org/1999/XMLSchema-instance'])"/>,
<xsl:value-of select="namespace-uri(namespace::*[string()='http://test/'])"/>
  </out>
</xsl:template>

</xsl:stylesheet>