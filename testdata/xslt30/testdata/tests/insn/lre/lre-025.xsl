<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- Type error: attempt to write a map to the content of an element.
    The real purpose of this test is to ensure that the error can be detected
    statically, though we can't actually guarantee that. -->

<xsl:template name="xsl:initial-template">
 <root>
   <a/>
   <xsl:sequence select="map{1:true(), 0:false()}"/>
 </root>
</xsl:template>

</xsl:stylesheet>
