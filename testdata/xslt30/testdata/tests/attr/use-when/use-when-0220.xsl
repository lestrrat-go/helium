<?xml version="1.0"?>
<t:transform version="2.0" xmlns:t="http://www.w3.org/1999/XSL/Transform" >
	
	<!-- Purpose: Test case that uses use-when=false() on the on the t:include but the 
		included stylesheet has a use-when=true().  -->
	
	<t:include href="include3.xsl" use-when="false()"/>
	
	<t:template match="doc">
		<out>
			<t:apply-templates />
		</out>
	</t:template>
	
	<t:template match="a" use-when="true()">
		<print_a><t:next-match/></print_a>
	</t:template>
	
</t:transform>