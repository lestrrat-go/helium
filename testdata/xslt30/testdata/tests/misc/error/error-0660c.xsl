<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a  
                        package contains more than one template with the same name and the same
                        import precedence, unless it
                     also contains a template with the same
                     name and higher import
                     precedence.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:template - duplicate name -->
<?spec xslt#errors?><?error XTSE0660?>


   <xsl:template xmlns:x="http://one.uri/" name="x:one"/>

  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  
   <xsl:template xmlns:y="http://one.uri/" name="y:one"/>

</xsl:stylesheet>
