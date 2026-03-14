<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
        <!--It is a non-recoverable dynamic error if the value of
            $input contains an escaped representation of a character (or codepoint)
            that is not a valid character in the version of XML supported by the implementation,
            unless the unescape option is set to false.-->
   <xsl:template name="main">
      <out>
         <xsl:variable name="j">[1,2,"xx\u0000xx"]</xsl:variable>
         <xsl:sequence select="json-to-xml($j)"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
