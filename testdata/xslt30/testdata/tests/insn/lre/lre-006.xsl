<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:ped="http://tester.com">

<xsl:import href="implre05.xsl"/>
<xsl:include href="inclre05.xsl"/>

<?spec xslt#lre-namespaces?>
    <!-- Purpose: Show that exclude-result-prefixes is scoped to just it's LRE. -->

  <!-- The designation of a namespace as an excluded namespace is 
       effective within the subtree of the stylesheet rooted at the element 
       bearing the exclude-result-prefixes or xsl:exclude-result-prefixes attribute.
       A subtree rooted at an xsl:stylesheet element does not include any 
       stylesheets imported or included by children of that xsl:stylesheet element. -->

  <!-- The ped namespace is excluded from main and foo, but not from the sub-elements that
    got placed in main by other templates. In fact, it has to be re-declared for each of
    those sub-elements. -->

<xsl:template match="docs">
  <main xsl:exclude-result-prefixes="ped">
     <foo/>
     <xsl:apply-templates/>
  </main>
</xsl:template>

<xsl:template match="doc1">
  <sub-element-in-main />
</xsl:template>
 
</xsl:stylesheet>
