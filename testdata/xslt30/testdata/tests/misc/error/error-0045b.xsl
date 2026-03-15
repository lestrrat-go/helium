<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
   version="3.0" 
   default-mode="does-not-exist">

                    <!--                       
                       Tests several variants of invocation that relate to throwing or not throwing the 
                       following error. If the invocation specifies #default, it takes whatever is in @default-mode
                       However, in an xsl:package, this default mode must exist or will raise XTSE3085 
                       (this error comes before XTDE0045, which is not applicable here, see bug #30221)
                    -->
                    
   <xsl:template match="/" mode="main">
      <out />
   </xsl:template>
</xsl:package>

