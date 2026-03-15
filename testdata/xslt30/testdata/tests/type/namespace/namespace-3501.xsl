<?xml version="1.0"?>
<ped:stylesheet xmlns:ped="http://www.w3.org/1999/XSL/Transform" version="2.0">
<ped:output method="xml"/>
<ped:key name="tk" match="xyz" use="zyx"/>
<ped:variable name="joe">2</ped:variable>
<ped:param name="sam" select="' x 4'"/>

<?spec xslt#xslt-namespace?>
    <!-- Purpose: Stylesheets are free to use any prefix, provided there is a namespace
       declaration that binds the prefix to the URI of XSLT namespace. -->

<ped:template match="doc">
  <out>
	<ped:value-of select="'Testing '"/>
	<ped:for-each select="*">
		<ped:value-of select="."/><ped:text> </ped:text>		
	</ped:for-each>

	<ped:text>&#010;</ped:text>
	<ped:value-of select="$joe"/>
	<ped:value-of select="$sam"/>

	<ped:text>&#010;</ped:text>
	<ped:call-template name="ThatTemp">
		<ped:with-param name="sam">quos</ped:with-param>
	</ped:call-template>

	<ped:text>&#010;</ped:text>
	<ped:call-template name="ThatTemp"/>
  </out>
</ped:template>

<ped:template name="ThatTemp">
	<ped:param name="sam">bo</ped:param>
	<ped:copy-of select="$sam"/>
</ped:template>

</ped:stylesheet>