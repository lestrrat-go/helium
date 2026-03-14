<?xml version="1.0"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<?spec xslt#applying-templates?>
<!-- BUG: proved not to be a bug at all -->


<xsl:template match="/credits_and_about_author/credits">
<HTML><HEAD><TITLE>About Author and Credits</TITLE></HEAD>
<BODY>
<H2>Credits</H2>
<xsl:text>
</xsl:text>

<TABLE>
<TR><TD><B>Acquisitions Editor</B><xsl:text>: </xsl:text></TD>
    <TD><xsl:apply-templates select="editor/acquisitions"/></TD>
</TR>
<TR><TD><B>Development Editor</B><xsl:text>: </xsl:text></TD>
    <TD><xsl:apply-templates select="editor/development"/></TD>
</TR>
<TR><TD><B>Contributing Writer</B><xsl:text>: </xsl:text></TD>
    <TD><xsl:apply-templates select="contributing"/></TD>
</TR>
<TR><TD><B>Technical Editor</B><xsl:text>: </xsl:text></TD>
    <TD><xsl:apply-templates select="editor/technical"/></TD>
</TR>
<xsl:for-each select="editor">
<TR>
    <TD><B>Copy Editors</B><xsl:text>: </xsl:text></TD>
    <TD><xsl:apply-templates select="copy_editor"/></TD>
</TR>
</xsl:for-each>
<TR>
    <TD><B>Production</B><xsl:text>: </xsl:text></TD>
    <TD><xsl:apply-templates select="production"/></TD>
</TR>
<TR><TD><B>Proofing and Indexing</B><xsl:text>: </xsl:text></TD>
    <TD><xsl:apply-templates select="proofing_indexing"/></TD>
</TR>
</TABLE>

<H2>About the Author</H2><P/>
Elliotte Rusty Harold is an internationally respected writer, programmer, and educator both on the Internet and off. He got his start by writing FAQ lists for the
Macintosh newsgroups on Usenet, and has since branched out into books, Web sites, and newsletters. He lectures about Java and object-oriented programming at
Polytechnic University in Brooklyn. His Cafe con Leche Web site at
<A HREF="http://metalab.unc.edu/xml/">http://metalab.unc.edu/xml/</A> has become one of the most popular independent XML sites on the
Internet. <P/>
Elliotte is originally from New Orleans where he returns periodically in search of a decent bowl of gumbo However, he currently resides in the Prospect Heights
neighborhood of Brooklyn with his wife Beth and cats Charm (named after the quark) and Marjorie (named after his mother-in-law). When not writing books, he
enjoys working on genealogy, mathematics, and quantum mechanics. His previous books include <I>The Java Developer's Resource, Java Network Programming,
Java Secrets, JavaBeans, XML: Extensible Markup Language, and Java I/O.</I>

</BODY>
</HTML>
</xsl:template>
</xsl:transform>
