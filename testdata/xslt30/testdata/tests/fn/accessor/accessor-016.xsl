<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:string accessor with empty sequence as argument. -->

   <xslt:template match="/">
	     <out>
         <xslt:variable name="temp1" select="string(())"/>
         <xslt:variable name="temp2" select="string(doc/inner/*)"/>
         <t1>
            <xslt:value-of select="$temp1 instance of xs:string and string-length($temp1) eq 0"/>
         </t1>
         <t2>
            <xslt:value-of select="$temp2 eq ''"/>
         </t2>
      </out>
	  </xslt:template>
</xslt:transform>
