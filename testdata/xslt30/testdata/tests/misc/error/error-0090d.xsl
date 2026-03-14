<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error for an element
                     from the XSLT namespace to have an attribute whose namespace is either null
                     (that is, an attribute with an unprefixed name) or the XSLT namespace, other
                     than attributes defined for the element in this document.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:call-template with surplus attribute -->
<?spec xslt#errors?><?error XTSE0090?>

  <xsl:template match="doc" name="this">
      <out>
         <xsl:call-template name="that" extra="spare">
            <xsl:with-param name="one" select="."/>
         </xsl:call-template> 
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  
  <xsl:template name="that">
      <xsl:param name="one"/>
  </xsl:template>


</xsl:stylesheet>
