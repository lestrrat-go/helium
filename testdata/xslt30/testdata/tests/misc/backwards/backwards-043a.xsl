<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
version="1.0">

     <xsl:key name="bob" match="*[@marker='a']" use="1"/>

     <xsl:template match="/">
         <result>
             <xsl:copy-of select="key('bob', 1)"/>
         </result>
     </xsl:template>

</xsl:stylesheet>
