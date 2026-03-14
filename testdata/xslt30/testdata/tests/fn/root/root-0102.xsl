<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Function is invoked with a element node -->

   <xslt:template match="/">
	 	   <out>
         <xslt:value-of select="root( doc/string1 )"/>
         <xslt:value-of select="root( doc/string1/xs:a )"/>
      </out>
	  </xslt:template>
</xslt:transform>
