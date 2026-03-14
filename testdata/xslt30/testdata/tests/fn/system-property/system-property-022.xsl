<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that system-property cannot be called in xsl:evaluate.-->
   
   <xslt:variable name="expr">system-property('Q{http://www.w3.org/1999/XSL/Transform}version')</xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:evaluate xpath="$expr"/>
      </out>
   </xslt:template>
</xslt:transform>
