<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                        processor is not able to retrieve the resource identified by the URI
                           reference in the href attribute of
                              xsl:include or xsl:import
                        , or if the resource that is retrieved does not contain a
                        stylesheet module conforming to this specification.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:import - target is not a stylesheet -->
<?spec xslt#errors?><?error?>

   <xsl:import href="_error-code-test-set.xml"/>
  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
