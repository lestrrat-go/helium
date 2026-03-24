<?xml version="1.0"?>
<jad:transform xmlns:jad="http://www.w3.org/1999/XSL/Transform" version="2.0">
<jad:output method="xml"/>

  <!-- FileName: incnspc13 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.1 XSLT Namespace -->
  <!-- Purpose: This stylesheet is being included by namespace13 which has the namespace
       prefix set to 'jad'. Testing that this setup with two different namespaces
       is allowed.  -->

<jad:template match="doc">
  <out>
	<jad:value-of select="'In Include: Testing '"/>
	<jad:for-each select="*">
		<jad:value-of select="."/><jad:text> </jad:text>		
	</jad:for-each>

	<jad:text>&#010;&#013;</jad:text>
	<jad:call-template name="ThatTemp">
		<jad:with-param name="sam">quos</jad:with-param>
	</jad:call-template>

  </out>
</jad:template>

<jad:template name="ThatOtherTemp">
  <jad:param name="sam">bo</jad:param>
	Included xmlns:jad <jad:copy-of select="$sam"/>
</jad:template>

</jad:transform>