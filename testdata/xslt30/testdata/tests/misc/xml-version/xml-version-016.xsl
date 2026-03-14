<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
<!-- Purpose: Tests constructing an xs:qname where the string literal is an XML 1.1 NCNAME. -->

   <xslt:output method="xml" version="1.1"/>

   <xslt:variable xmlns:pre="http://example.org/preÀ"
                  name="var"
                  as="xs:QName"
                  select="xs:QName('pre:Ĭ')"/>

   <xslt:template match="/">
		    <out>
         <xslt:value-of select="$var"/>
      </out>
	  </xslt:template>
</xslt:transform>
