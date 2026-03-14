<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                 <!--It is a recoverable dynamic error
                     for a transformation to generate two or more final result trees with URIs that
                     identify the same physical resource. The optional recovery action is
                     implementation-dependent, since it may be impossible for the
                     processor to detect the error.-->
                     
   <xsl:template name="main">
      <out>
         <xsl:message terminate="1" error-code="Q{{http://www.w3.org/2005/xqt-errors}}XTDE1495">Error not testable in an interoperable way</xsl:message>      
      </out>
   </xsl:template>
</xsl:stylesheet>
