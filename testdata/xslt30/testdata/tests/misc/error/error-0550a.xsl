<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the list
                        of modes in the mode attribute of
                              xsl:template
                         is empty, if the same token is included more than once in the
                        list, if the list contains an invalid token, or if the token
                           #all appears together with any other value.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:template - bad mode name -->
<?spec xslt#errors?><?error XTSE0280?>

   <xsl:template match="p" mode="name!1223"/>
  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
