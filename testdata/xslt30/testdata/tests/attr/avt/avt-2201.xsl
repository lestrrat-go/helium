<?xml version="1.0" encoding="iso-8859-1"?>

<xsl:stylesheet version="2.0"

xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#xsl-element?>



<xsl:output method="xml"/>

    <xsl:variable name="str" select="'str'"/>

<xsl:template match="/">

  <erg>

    <!-- this one works -->

    <aa name="{$str}"/>

    <!-- @@1 this one gives "not a QName" error -->

    <bb><xsl:attribute name="{$str}"/></bb>

    <!-- @@2 this one gives "not a QName" error -->

    <xsl:element name="{$str}"/>

    <!-- this one works -->

    <xsl:element name="a{$str}"/>


    <xsl:variable name="rtf">rtf</xsl:variable>

    <!-- this one works -->

    <aa name="{$rtf}"/>

    <!-- this one works -->

    <bb><xsl:attribute name="{$rtf}"/></bb>

    <!-- this one works -->

    <xsl:element name="{$rtf}"/>

  </erg>

</xsl:template>



</xsl:stylesheet>
