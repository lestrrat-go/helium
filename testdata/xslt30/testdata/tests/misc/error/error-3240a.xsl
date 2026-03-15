<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
        <!--It is a non-recoverable dynamic error if the value of
            $input does not conform to the JSON grammar, as selected using the
            explicit or implicit spec option.-->
   <xsl:template name="main">
      <out>
         <xsl:variable name="j">[1,2,3x]</xsl:variable>
         <xsl:sequence select="json-to-xml($j)"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
