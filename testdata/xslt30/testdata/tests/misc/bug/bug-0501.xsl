<?xml version="1.0" ?>

<!-- BUG: Saxon 5.2/021  -->
<!-- DESCRIPTION: Every element should have a namespace node for the "xml" namespace. It doesn't.-->
<?spec dm#NamespaceNode?>
<?same-as-1.0 no?>

<out xsl:version="2.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<xsl:for-each select="*/namespace::*">
  <xsl:sort select="(index-of(('xml', 'xsl', 'axsl'), name()), 99)[1]"/> <!-- for reproducible results --> 
  <xsl:value-of select="name()"/>=<xsl:value-of select="."/>;
</xsl:for-each>

</out>
