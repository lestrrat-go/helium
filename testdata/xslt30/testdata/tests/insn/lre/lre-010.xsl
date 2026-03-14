<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
	xmlns:ext="http://somebody.elses.extension"
      extension-element-prefixes="ext">

<?spec xslt#attributes-for-lres?>
    <!-- Purpose: Show that attributes from extension namespaces cause inclusion of the namespace node. -->
  <!-- Purpose: The created element node will also have a copy of the namespace nodes 
       that were present on the element node in the stylesheet tree with the exception 
       of any namespace node whose string-value is the XSLT namespace URI, a namespace 
       URI declared as an extension namespace, or a namespace URI designated as an 
       excluded namespace. -->

<xsl:template match="doc">
  <out ext:size="big" english="to leave"/>
</xsl:template>

</xsl:stylesheet>