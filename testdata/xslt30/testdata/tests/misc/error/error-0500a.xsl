<?xml version="1.0" encoding="UTF-8"?>
<!--An xsl:template element must have either a
                        match attribute or a name attribute, or both. An
                        xsl:template element that has no match
                     attribute must have no mode attribute and no
                        priority attribute. An xsl:template
                        element that has no name attribute must have no visibility
                        attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:template - no match or name -->
<?spec xslt#errors?><?error XTSE0500?>

   <xsl:template mode="x"/>
  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
