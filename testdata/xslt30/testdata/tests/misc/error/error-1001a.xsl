<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--It is a dynamic error if the effective value
                  of the start-at attribute of the xsl:number
                  instruction is not in the lexical space of xs:integer. The error may be signaled
                  statically if it can be detected statically.-->
                  
   <xsl:template name="main">
      <out>
         <xsl:number value="78" start-at="alpha"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
