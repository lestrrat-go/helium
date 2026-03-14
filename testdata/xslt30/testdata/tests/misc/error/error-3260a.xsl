<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"><!--It is a non-recoverable dynamic
               error if the value of
            $options includes an entry whose key is "liberal" and whose value is not
            a single xs:boolean, or an entry whose key is "escape" and whose value
            is not a single xs:boolean.-->
   <xsl:template name="main">
      <out>
         <xsl:sequence select="json-to-xml('[1,2,3]', map{'liberal':'yes'})"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
