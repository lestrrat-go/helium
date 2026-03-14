<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                    <!--It is a non-recoverable dynamic
                        error if the arguments supplied to a call on an extension
                        function do not satisfy the rules defined for that particular extension
                        function, or if the extension function reports an error, or if the result of
                        the extension function cannot be converted to an XPath value.-->
                        
   <xsl:template name="main">
      <out>
         <xsl:message terminate="1" error-code="Q{{http://www.w3.org/2005/xqt-errors}}XTDE1420">Error not testable in an interoperable way</xsl:message>
      </out>
   </xsl:template>
</xsl:stylesheet>
