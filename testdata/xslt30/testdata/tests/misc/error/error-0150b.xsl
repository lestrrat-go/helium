<?xml version="1.0" encoding="UTF-8"?>
                 <!--A literal result element
                     that is used as the outermost element of a simplified stylesheet module
                     must have an xsl:version attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:include - target is not a stylesheet -->
<?spec xslt#errors?><?error?>


  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  
   <xsl:include href="_error-test-set.xml"/>

</xsl:stylesheet>
