<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="3.0">
	<!-- 
		Purpose: Test case using a variable with static="true" 
		variant of use-when-0411, with a valid static var and a body with use-when=false
	-->

   <xslt:variable name="x" as="xs:string" select="'correct'" static="true">
      <!-- static variables may not have a body, but seqtor is disabled, so it is allowed -->
   	<not-allowed-but-disabled xslt:use-when="system-property('xslt:version') = '10.1'" />
   </xslt:variable>

   <xslt:template match="/">   
      <out var="{$x}" />      
   </xslt:template>
</xslt:transform>
