<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#attributes-for-lres?>
<?error XTSE0085?>


    <!-- Purpose: Show that attributes from XSLT namespace are automatically excluded. -->
  <!-- Elaboration: The created element node will also have a copy of the namespace nodes 
       that were present on the element node in the stylesheet tree with the exception 
       of any namespace node whose string-value is the XSLT namespace URI, a namespace 
       URI declared as an extension namespace, or a namespace URI designated as an 
       excluded namespace. -->

  <!-- MHK comment: Saxon disallows this. I can't actually see anything in the XSLT 1.0
       specification that bans it. It's banned in 2.0 -->

<xsl:template match="doc">
  <out xsl:if="my if" english="to leave"/>
</xsl:template>

</xsl:stylesheet>