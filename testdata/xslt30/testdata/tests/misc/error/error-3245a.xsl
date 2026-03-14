<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
        <!--Error: It is a dynamic error if the value of the validate option is true 
            and the processor is not schema-aware.-->
   <xsl:template name="main">
      <out>
         <xsl:variable name="j">[1,2,3]</xsl:variable>
         <xsl:sequence select="json-to-xml($j, map{'validate':true()})"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
