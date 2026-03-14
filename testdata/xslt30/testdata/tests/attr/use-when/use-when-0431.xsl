<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="3.0">
	<!-- 
		Purpose: Test case using a variable with static="true" and error 
		variant of use-when-0421, with use-when in no-namespace on an LRE, should raise error
	-->

   <xslt:variable name="x" as="xs:string" select="'correct'" static="true">
      <!-- static variables may not have a body, seqtor is incorrectly disabled (use-when not in xsl namespace), not allowed -->
   	<not-allowed-but-disabled use-when="system-property('xslt:version') = '10.1'" />
   </xslt:variable>

   <xslt:template match="/">   
      <out var="{$x}" />      
   </xslt:template>
</xslt:transform>
